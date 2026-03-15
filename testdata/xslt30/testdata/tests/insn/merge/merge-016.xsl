<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs"
    version="3.0">
    <xsl:output method="xml" indent="no"/>
    <xsl:template match="/">
        
        <results>
            <xsl:merge>
                <xsl:merge-source select="doc('school1.xml')/school/class[@id='error']/pupil">
                        <xsl:merge-key select="number(tests/test[@id='test1']/mark)" order="descending"/>
                </xsl:merge-source>
                <xsl:merge-action>
                    <xsl:value-of select="current-merge-group()"/>                    
                </xsl:merge-action>
            </xsl:merge>
        </results>
        
    </xsl:template>
</xsl:stylesheet>