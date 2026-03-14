<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
        xmlns:xs="http://www.w3.org/2001/XMLSchema"
        xmlns="http://schema068.uri/"
        xmlns:t="http://schema068.uri/"
        exclude-result-prefixes="xs t">

<!-- PURPOSE: Test validation failure on xsl:result-document instruction using the type attribute -->
              <?spec xslt#validation-xsl-type?>
              <?error ?>

<xsl:import-schema schema-location="schema068.xsd" 
     namespace="http://schema068.uri/"/>
  



  <xsl:template match="/">
    <xsl:result-document type="t:outType">
    <out>
      <pperson id="p1">           <!-- wrong -->
        <first>Michael</first>
        <middle>H</middle>
        <last>Kay</last>
      </pperson>
      <person id="p2">
        <first>Penny</first>
        <surname>Whelan</surname>
      </person>
      <person id="p3">
        <last>Kay</last>
        <first>Michael</first>
        <middle>H</middle>
      </person> 
      <person xsl:validation="strict">
        <last>Kay</last>
        <first>Michael</first>
      </person>        
    </out>
    </xsl:result-document>
  </xsl:template>
  
</xsl:stylesheet>