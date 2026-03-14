<?xml version="1.0"?> 

<?spec xslt#stylesheet-functions?>
<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:saxon="http://NOT-REALLY.saxon.sf.net/"
  xmlns:sql="http://NOT-REALLY.net.sf.saxon.sql.SQLElementFactory/"
  xmlns:other="http://nowhere.uri/" exclude-result-prefixes="saxon sql other">

  <!-- Test element-available on a range of names -->
  <!-- Originally a Saxon test in which some of these extensions were available;
      now made interoperable (and somewhat pointless) by changing the namespaces -->
  <?spec xslt#testing-instruction-available?>


  <xsl:template match="/">
    <xsl:param name="xsl">xsl:</xsl:param>
    <xsl:param name="zls" select="''"/>
    <out>
      <a1 actual="{element-available('xsl:value-of')}" expected="true"/>
      <a2 actual="{element-available('saxon:assign')}" expected="false"/>
      <a3 actual="{element-available('sql:connect')}" expected="false"/>
      <a4 actual="{element-available('xsl:key')}" expected="true"/>
      <a5 actual="{element-available('xsl:neverHeardOfIt')}" expected="false"/>
      <a6 actual="{element-available('saxon:collation')}" expected="false"/>
      <a7 actual="{element-available('other:handler')}" expected="false"/>
      <a8 actual="{element-available('get-a-life')}" expected="false"/>
      <a9 actual="{element-available(concat($xsl, 'text'))}" expected="true"/>
      <variable name="v10" xmlns="http://www.w3.org/1999/XSL/Transform" select="element-available($zls || 'value-of')"/> 
      <a10 actual="{$v10}" expected="true"/>
    </out>
  </xsl:template>

</xsl:stylesheet>
