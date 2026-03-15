<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema" 
    xmlns:math="http://www.w3.org/2005/xpath-functions/math" exclude-result-prefixes="xs math" 
    version="3.0">
    
    <xsl:param name="STREAMABLE" static="yes" as="xs:boolean" select="true()"/>
    
    <xsl:mode _streamable="{$STREAMABLE}"/>
    
    <xsl:template match="Root">
        <xsl:copy>
            <xsl:fork>
                <xsl:for-each-group select="GROUP/text()[contains(., ',TYPE,')]" 
                    group-by="substring-after(., ',TYPE,')">
                    <Brand>
                        <Vehicle>
                            <xsl:value-of select="current-grouping-key()"/>
                        </Vehicle>
                    </Brand>
                </xsl:for-each-group>
            </xsl:fork>
        </xsl:copy>
    </xsl:template>
    
</xsl:stylesheet>