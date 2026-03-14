<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">
<?spec fo#func-avg?>
<!-- PURPOSE:  Test that avg() returns the same type as its argument -->

<xsl:param name="all" select="//a"/>
<xsl:param name="int" select="for $x in $all return xs:integer($x)"/>
<xsl:param name="dec" select="for $x in $all return xs:decimal($x)"/>
<xsl:param name="flt" select="for $x in $all return xs:float($x)"/>
<xsl:param name="dbl" select="for $x in $all return xs:double($x)"/>

<xsl:template match="doc">
<r>
<out test="int" check="{$int instance of xs:integer*}">
<a val="{avg($int)}" check="{avg($int) instance of xs:decimal}"/>;
</out>
<out test="dec" check="{$dec instance of xs:decimal*}">
<a val="{avg($dec)}" check="{avg($dec) instance of xs:decimal}"/>;
</out>
<out test="flt" check="{$flt instance of xs:float*}">
<a val="{avg($flt)}" check="{avg($flt) instance of xs:float}"/>;
</out>
<out test="dbl" check="{$dbl instance of xs:double*}">
<a val="{avg($dbl)}" check="{avg($dbl) instance of xs:double}"/>;
</out>
</r>
</xsl:template>
</xsl:stylesheet>