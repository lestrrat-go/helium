<?xml version="1.0" encoding="UTF-8"?>
<xsl:package xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:fo="http://www.w3.org/1999/XSL/Format"
    xmlns:ahf="http://www.antennahouse.com/Names/XSLT/Document"
    package-version="1.0"
    name="urn:170c"
    exclude-result-prefixes="xs ahf"
    version="3.0"
    default-mode="normal">
    
    <xsl:mode name="normal" visibility="public"/>
    <xsl:mode/>
    
    <xsl:template match="*[contains(@class,' topic/ph ')]" name="tplPh">
        <fo:inline matched-by="tplPh">
            <xsl:apply-templates/>
        </fo:inline>
    </xsl:template>
    
    <xsl:template match="*[contains(@class,' hi-d/b ')]" priority="2" name="tplB">
        <fo:inline font-weight="bold">
            <xsl:apply-templates/>
        </fo:inline>
    </xsl:template>
    
    <xsl:function name="ahf:setRed" as="attribute()*" visibility="public">
        <xsl:attribute name="color" select="'red'"/>
    </xsl:function>
    
</xsl:package>
