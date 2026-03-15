<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="xs">
<?spec fo#casting-to-numerics?>
<!-- PURPOSE:  Test that casting to integer truncates towards zero. -->

<xsl:template match="doc">
<out>
<decimal a="{xs:integer(3.8)}" b="{xs:integer(-3.8)}"/>
<double a="{xs:integer(3.8e0)}" b="{xs:integer(-3.8e0)}"/>
<float a="{xs:integer(xs:float(3.8e0))}" b="{xs:integer(xs:float(-3.8e0))}"/>
</out>
</xsl:template>
</xsl:stylesheet>