<?xml version="1.0" encoding="UTF-8"?>
<xsl:package xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:math="http://www.w3.org/2005/xpath-functions/math"
    exclude-result-prefixes="xs math"
    version="3.0"
    name="urn:176a">
    
    <xsl:use-package name="urn:176b"/>
    <xsl:use-package name="urn:176c"/>
    <xsl:mode/>
    
    <xsl:template name="xsl:initial-template" visibility="public">
        <out>
            <bb><xsl:call-template name="b"/></bb>
            <cc><xsl:call-template name="c"/></cc>
        </out>
    </xsl:template>
    
</xsl:package>
