<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema" 
    xmlns:math="http://www.w3.org/2005/xpath-functions/math" 
    exclude-result-prefixes="xs math" 
    version="3.0">
    
    <xsl:template name="xsl:initial-template">
            <data>
                <xsl:source-document href="si-iterate-139.xml" streamable="true">
                    <xsl:iterate select="snapshot(//a)">
                        <xsl:param name="count" select="1"/>
                        <out c="{$count}"/>
                        <xsl:apply-templates select="."/>
                        <xsl:choose>
                            <xsl:when test="$count le 100">
                                <xsl:next-iteration>
                                    <xsl:with-param name="count" select="$count + 1"/>
                                </xsl:next-iteration>
                            </xsl:when>
                            <xsl:otherwise>
                                <xsl:break/>
                            </xsl:otherwise>
                        </xsl:choose>
                    </xsl:iterate>
                </xsl:source-document>
            </data>

    </xsl:template>
    
</xsl:stylesheet>