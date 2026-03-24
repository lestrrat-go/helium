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
        <xsl:accept component="mode" names="normal" visibility="public" />
    </xsl:use-package>
    
    <xsl:mode/>
    
    <xsl:template match="/*">
        <fo:block>
            <xsl:copy-of select="ahf:setRed()"/>
            <xsl:apply-templates mode="normal"/>
        </fo:block>
    </xsl:template>
    
</xsl:package>
