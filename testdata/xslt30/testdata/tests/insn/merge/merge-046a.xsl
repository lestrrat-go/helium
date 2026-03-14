<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs"
    version="3.0">
    
    <!-- testing invalid name: XTSE0020 -->
    <xsl:output method="xml" indent="no"/>
    <xsl:template name="xsl:initial-template" visibility="public">
        <events>
            <xsl:merge>
                <xsl:merge-source 
                	name="one"
                	select="doc('log-file-1.xml')/events/event">
                        <xsl:merge-key select="xs:dateTime(@timestamp)"/>
                </xsl:merge-source>
                <xsl:merge-source 
                	name="...."
                	select="doc('log-file-2.xml')/log/day/record">
                        <xsl:merge-key select="dateTime(../@date, time)"/>
                </xsl:merge-source>
                <xsl:merge-action>
                    <group at="{current-merge-key()}">
                        <one><xsl:copy-of select="current-merge-group('one')" /></one>
                    </group>
                </xsl:merge-action>
            </xsl:merge>
        </events>
        
    </xsl:template>
</xsl:stylesheet>