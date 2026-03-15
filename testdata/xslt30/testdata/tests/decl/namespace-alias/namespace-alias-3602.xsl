<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
  xmlns:bogus="http://www.bogus_ns.com" xmlns:lotus="http://www.lotus.com"
  xmlns:ped="http://www.ped.com/">

  <?spec xslt#extension-attributes?>
  <!-- Purpose:  Testing an attribute not from the XSLT namespace, which is
       legal provided that the expanded name of the attribute has a non-null 
       namespace URI. This tests for many xslt elements, apparent code path 
       are different for numerous elements. Should actually output a "bogus"
       stylesheet. -->

  <xsl:import href="namespace-alias-3602a.xsl" ped:a="a"/>
  <xsl:include href="namespace-alias-3602b.xsl" ped:b="b"/>
  <xsl:output method="xml"  lotus:c="c"/>

  <xsl:key name="sprtest" match="TestID" use="Name" lotus:d="d"/>

  <xsl:strip-space elements="a" ped:e="e"/>
  <xsl:preserve-space elements="b" lotus:f="f"/>

  <xsl:variable name="Var1" ped:g="g">DefaultValueOfVar1</xsl:variable>

  <xsl:param name="Param1" lotus:h="h">DefaultValueOfParam1</xsl:param>

  <xsl:attribute-set name="my-style" ped:i="i">
    <xsl:attribute name="my-size" lotus:j="j">12pt</xsl:attribute>
    <xsl:attribute name="my-weight">bold</xsl:attribute>
  </xsl:attribute-set>

  <xsl:namespace-alias stylesheet-prefix="bogus" result-prefix="xsl" ped:k="k"/>
  <xsl:decimal-format decimal-separator="," grouping-separator=" " lotus:l="l"/>

  <xsl:template match="/" priority="2">
    <bogus:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="1.0">
      <bogus:template match="/">
        <out>Yeee ha</out>
      </bogus:template>
    </bogus:stylesheet>
  </xsl:template>

</xsl:stylesheet>
