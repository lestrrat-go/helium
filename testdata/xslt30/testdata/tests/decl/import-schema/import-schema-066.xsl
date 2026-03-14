<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
        xmlns:xs="http://www.w3.org/2001/XMLSchema"
        xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
        xmlns:z="http://schema076.uri/"
        exclude-result-prefixes="xsi">

<!-- PURPOSE: Test that string-length() and normalize-space with no arguments use
     the string value not the typed value -->
     <?spec fo#func-string-length?>
     <?spec fo#func-normalize-space?>
     <?spec xpath#id-typed-value?>

<xsl:import-schema namespace="http://schema076.uri/" schema-location="schema076.xsd"/>



  <xsl:template match="/">
    <xsl:variable name="x" as="element()">
        <n xsl:type="z:numbers">1.000 2.000 3.000 4.000</n>
    </xsl:variable>
    <out>
      <xsl:for-each select="$x">
          <l><xsl:value-of select="string-length()"/></l>
          <s><xsl:value-of select="normalize-space()"/></s>
      </xsl:for-each>
    </out>
  </xsl:template>
  
</xsl:stylesheet>