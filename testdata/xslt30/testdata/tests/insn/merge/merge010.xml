<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs"
    version="3.0">
    
    <!-- xsl:merge test of intersection: -->
    <xsl:output method="xml" indent="no"/>
    <xsl:strip-space elements="*"/>
    <xsl:template match="/">
        
        <results>
            <xsl:merge>
                <xsl:merge-source select="1 to 30">
                        <xsl:merge-key select="."/>
                </xsl:merge-source>
                <xsl:merge-source select="20 to 40">
                        <xsl:merge-key select="."/>
                </xsl:merge-source>
                <xsl:merge-action>
                    <xsl:if test="count(current-group()) eq 2">
                        <out>
                        <xsl:value-of select="current-grouping-key()"/>
                        </out>
                    </xsl:if>
                </xsl:merge-action>
            </xsl:merge>
        </results>
        
    </xsl:template>
</xsl:stylesheet>