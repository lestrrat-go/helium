<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
        xmlns:xs="http://www.w3.org/2001/XMLSchema"
        xmlns="http://schema067.uri/"
        xmlns:a="http://schema067.uri/"
        exclude-result-prefixes="xs">

<!-- PURPOSE: Test use of a schema that imports another schema. This schema also mixes
qualified and unqualified attribute forms -->
              <?spec xslt#import-schema?>
              <?spec xslt#validating-using-validation-attribute?>

<xsl:import-schema schema-location="schema067.xsd" 
     namespace="http://schema067.uri/"/>
  



  <xsl:template match="/">
    <address zip="rg4" a:color="red blue" xsl:validation="strict">
      <person>
        <first>Michael</first>
        <middle>H</middle>
        <last>Kay</last>
      </person>
      <age>52</age>
    </address>
  </xsl:template>
  
</xsl:stylesheet>