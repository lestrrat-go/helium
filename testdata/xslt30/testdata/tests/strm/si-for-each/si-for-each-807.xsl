<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema" 
    xmlns:math="http://www.w3.org/2005/xpath-functions/math" 
    exclude-result-prefixes="xs math" 
    version="3.0">
    
    <xsl:mode streamable="yes"/>
    
    <xsl:template match="/">
        <xsl:for-each select="snapshot(class/students/student)">
            <xsl:result-document href="student{format-number(position(), '000')}.xml">
                <xsl:copy-of select="/"/>
            </xsl:result-document>
        </xsl:for-each>
    </xsl:template>
    
</xsl:stylesheet>