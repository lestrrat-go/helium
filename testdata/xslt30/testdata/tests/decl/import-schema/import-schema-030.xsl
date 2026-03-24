<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
        xmlns:xs="http://www.w3.org/2001/XMLSchema"
        xmlns:xslxx="http://schema039.uri/"
        exclude-result-prefixes="xs xslxx">

<!-- PURPOSE: Test use of a used-defined list type: xsl:QNames -->


<xsl:import-schema schema-location="schema039.xsd" 
     namespace="http://schema039.uri/"/>


  <xsl:template match="/">
    <out>
      <xsl:variable name="t" as="attribute(*,xslxx:QNames)">
        <xsl:attribute name="a" type="xslxx:QNames" select="'red green hat:blue hat:pink'"/>
      </xsl:variable>
      <a><xsl:value-of select="string-join($t, ',')"/></a>
      <b><xsl:value-of select="$t[1] = 'green'"/></b>
      <c><xsl:value-of select="data($t) instance of xslxx:QName*"/></c>
      <d><xsl:value-of select="data($t) instance of xs:string*"/></d>
        
    </out>
  </xsl:template>
  
</xsl:stylesheet>