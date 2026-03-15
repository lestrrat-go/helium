<?xml version="1.0" ?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:f="http://local/" version="3.0"
    exclude-result-prefixes="#all" expand-text="yes">

    <xsl:template name="xsl:initial-template">
        <out>
            <t>{f:dyn('a b c') castable as xs:IDREFS}</t>
            <t>{f:dyn(' aa bb c123 ') castable as xs:IDREFS}</t>
            <t>{f:dyn(' aa bb c1.2 ') castable as xs:IDREFS}</t>
            <f>{f:dyn('1 2 3 ') castable as xs:IDREFS}</f>
            <f>{f:dyn(' 1/2 ') castable as xs:IDREFS}</f>
            <f>{f:dyn('') castable as xs:IDREFS}</f>
            <f>{f:dyn(' ') castable as xs:IDREFS}</f>
            <f>{f:dyn(' &#10; ') castable as xs:IDREFS}</f>
        </out>
    </xsl:template>
    
    <xsl:function name="f:dyn">
        <xsl:param name="in"/>
        <xsl:sequence select="$in[current-date() gt xs:date('1900-01-01')]"/>
    </xsl:function>
</xsl:stylesheet>
