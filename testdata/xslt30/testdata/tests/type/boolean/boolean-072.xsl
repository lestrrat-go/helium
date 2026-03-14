<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0" xmlns:f="http://f.com/"
exclude-result-prefixes="f xs"
xmlns:xs="http://www.w3.org/2001/XMLSchema">

<?spec xpath#id-value-comparisons?>
<!-- PURPOSE: Test of eq (etc) with an emptiable types as operands -->
<?same-as-1.0 no?>


<xsl:template match="/">
    <a><xsl:value-of select="f:eq((), ()), f:eq(1, ()), f:eq((), 'fred')" separator="|"/></a>
</xsl:template>

<xsl:function name="f:eq" as="xs:boolean?">
    <xsl:param name="a" as="xs:integer?"/>
    <xsl:param name="b" as="xs:string?"/>
    <xsl:sequence select="$a eq $b"/>
</xsl:function>

</xsl:stylesheet>
