<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema" 
    xmlns:math="http://www.w3.org/2005/xpath-functions/math" exclude-result-prefixes="xs math" 
    version="3.0">
    
    <xsl:strip-space elements="*"/>
    
    <xsl:mode streamable="yes" on-no-match="shallow-copy"/>
    
    <xsl:template match="books">
        <root>
            <subjects>
                <xsl:for-each-group select="copy-of(book)" group-by="subject">
                    <subject name="{current-grouping-key()}">
                        <xsl:apply-templates select="current-group()"/>
                    </subject>
                </xsl:for-each-group>
            </subjects>
        </root>
    </xsl:template>
    
    <xsl:template match="book/subject"/>
    
</xsl:stylesheet>