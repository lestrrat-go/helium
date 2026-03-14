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
    <xsl:param name="xsl">Q{http://www.w3.org/1999/XSL/Transform}</xsl:param>
    <xsl:param name="zls" select="'Q{}'"/>
    <out>
      <a1 actual="{element-available('Q{http://www.w3.org/1999/XSL/Transform}value-of')}" expected="true"/>
      <a2 actual="{element-available('Q{http://not-saxon.sf.net}assign')}" expected="false"/>
      <a4 actual="{element-available('Q{http://www.w3.org/1999/XSL/Transform}key')}" expected="true"/>
      <a5 actual="{element-available('Q{http://www.w3.org/1999/XSL/Transform}neverHeardOfIt')}" expected="false"/>
      <a8 actual="{element-available('Q{}get-a-life')}" expected="false"/>
      <a9 actual="{element-available(concat($xsl, 'text'))}" expected="true"/>
      <variable name="v10" xmlns="http://www.w3.org/1999/XSL/Transform" select="element-available($zls || 'value-of')"/> 
      <a10 actual="{$v10}" expected="false"/>
    </out>
  </xsl:template>

</xsl:stylesheet>
