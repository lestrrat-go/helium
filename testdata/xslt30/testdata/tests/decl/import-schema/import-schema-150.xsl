<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
        xmlns:xs="http://www.w3.org/2001/XMLSchema"
        xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
        xmlns:z="http://schema061.uri/"
        exclude-result-prefixes="xsi">

<!-- PURPOSE: Type-checking of path expressions against a schema -->

<xsl:import-schema namespace="http://schema061.uri/" schema-location="schema061.xsd"/>
             <?spec xpath#id-schema-element-test?>
              <?error ?>



<xsl:variable name="mhk" as="schema-element(z:person)">
    <person xsl:validation="strict" xmlns="http://schema061.uri/">
      <first>Michael</first>
      <middle>H</middle>
      <last>Kay</last>
      <father>
        <first>Leslie</first>
        <middle>R</middle>
        <last>Kay</last>
      </father>
    </person>
</xsl:variable>

  <xsl:template match="/">
    <out>
      <xsl:apply-templates select="$mhk"/>
    </out>
  </xsl:template>
  
  <xsl:template match="schema-element(z:person)">
    <h2><xsl:value-of select="z:first, z:middle, z:last"/></h2>
    <p>First name: <xsl:value-of select="z:first"/></p>
    <p>Father's first name: <xsl:value-of select="z:father/z:first[@diff=3]"/></p>
  </xsl:template>
  
  
</xsl:stylesheet>