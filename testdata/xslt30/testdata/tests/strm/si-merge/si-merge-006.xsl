<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs"
    version="3.0">
    
    <!-- Purpose: Test streamable=" true " and " 1 "; merge XML file with unparsed text file
        (based on example in spec). -->

    <xsl:template name="main">
        
        <events>
            <xsl:merge>
                <xsl:merge-source name="fax" 
                    select="unparsed-text-lines('../docs/fax-log.txt')">
                    <xsl:merge-key select="xs:date(substring-before(., ' '))"/>
                </xsl:merge-source>
                <xsl:merge-source name="transactions"
                    for-each-source="'../docs/transactions.xml'" 
                    select="/account/transaction" 
                    streamable="yes">
                    <xsl:merge-key select="xs:date(@date)"/>
                </xsl:merge-source>
                <xsl:merge-action>
                    <messages at="{current-merge-key()}">
                        <xsl:where-populated>
                            <fax>
                                <xsl:for-each select="current-merge-group('fax')">
                                    <message xsl:expand-text="true">{
                                        substring-after(., ' ')
                                        }</message>
                                </xsl:for-each>   
                            </fax>
                            <transaction>
                                <xsl:sequence select="current-merge-group('transactions')"/>
                            </transaction>
                        </xsl:where-populated>   
                    </messages>   
                </xsl:merge-action>
            </xsl:merge>
        </events>
        
    </xsl:template>
</xsl:stylesheet>