<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs"
    version="2.0">
    <!-- streamed xsl:for-each-group: current-grouping-key not available in applied template -->
    
    <xsl:template name="g-015">
        <out>
            <xsl:source-document streamable="yes" href="../docs/books-atts.xml">
                <xsl:fork>
                    <xsl:for-each-group select="/BOOKLIST/BOOKS/ITEM" group-by="@AUTHOR">
                        <g author="{current-grouping-key()}">
                            <xsl:apply-templates select="current-group()" mode="cgk"/>
                        </g>
                    </xsl:for-each-group>
                </xsl:fork>
            </xsl:source-document>
        </out>
    </xsl:template>
    
    <xsl:mode name="cgk" streamable="yes"/>
    
    <xsl:template match="." mode="cgk">
        <xsl:try>
            <h key="{current-grouping-key()}"/>
            <xsl:catch errors="*:XTDE1071">
                <h key="#absent#"/>
            </xsl:catch>
        </xsl:try>  
    </xsl:template>
</xsl:stylesheet>