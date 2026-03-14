<?xml version="1.0" encoding="UTF-8"?>
<xsl:package xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:math="http://www.w3.org/2005/xpath-functions/math"
    xmlns:fo="http://www.w3.org/1999/XSL/Format"
    xmlns:ahf="http://www.antennahouse.com/Names/XSLT/Document"
    exclude-result-prefixes="xs math ahf"
    version="3.0"
    name="urn:170a">
    
    <xsl:use-package name="urn:170b">
        <xsl:override>
            <xsl:template match="*[contains(@class,' topic/ph ')]" mode="normal">
                <fo:inline>
                    <xsl:text>two:</xsl:text>
                    <xsl:next-match/>
                </fo:inline>
            </xsl:template>
            <xsl:template match="*[contains(@class,'/ph ')]" mode="normal">
                <fo:inline>
                    <xsl:text>one:</xsl:text>
                    <xsl:next-match/>
                </fo:inline>
            </xsl:template>
        </xsl:override>
    </xsl:use-package>
    
    <xsl:mode/>
    
    <xsl:template match="/*">
        <fo:block>
            <xsl:copy-of select="ahf:setRed()"/>
            <xsl:apply-templates mode="normal"/>
        </fo:block>
    </xsl:template>
    
</xsl:package>
