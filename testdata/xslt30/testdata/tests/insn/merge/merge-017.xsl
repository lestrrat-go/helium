<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs"
    version="3.0">
    <!-- An error should be raised as a sequence of more than one item is not allowed as 
        the @select attribute of xsl:merge-key -->
    
    <xsl:output method="xml" indent="no"/>
    <xsl:template match="/">
        
        <results>
            <xsl:merge>
                <xsl:merge-source select="doc('school1.xml')/school/class/pupil, doc('school2.xml')/school/class/pupil">
                   
                        <xsl:merge-key select="tests/test/mark, tests/test/mark" order="descending"/>
                    
                </xsl:merge-source>
                <xsl:merge-action>
                    <xsl:copy-of select="current-merge-group()"/>                    
                </xsl:merge-action>
            </xsl:merge>
        </results>
        
    </xsl:template>
</xsl:stylesheet>