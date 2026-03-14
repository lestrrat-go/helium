<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
        xmlns:xs="http://www.w3.org/2001/XMLSchema"
        xmlns="http://schema068.uri/"
        xmlns:t="http://schema068.uri/"
        exclude-result-prefixes="xs t">

<!-- PURPOSE: Test validation error at document level with xsl:copy-of instruction using the type attribute -->
              <?spec xslt#validation-xsl-type?>
              <?error ?>

<xsl:import-schema schema-location="schema068.xsd" 
     namespace="http://schema068.uri/"/>
  



  <xsl:template match="/">
    <xsl:variable name="n">
    <out>
      <person id="p1">
        <first>Michael</first>
        <middle>H</middle>
        <last>Kay</last>
      </person>
      <person id="p2">
        <first>Penny</first>
        <surname>Whelan</surname>
      </person>
      <person pid="p3">
        <last>Kay</last>
        <first>Michael</first>
        <middle>H</middle>
      </person> 
    </out>
    </xsl:variable>
    <xsl:copy-of select="$n" type="t:outType"/>
  </xsl:template>
  
</xsl:stylesheet>