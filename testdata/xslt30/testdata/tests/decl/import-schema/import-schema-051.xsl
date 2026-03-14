<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
        xmlns:xs="http://www.w3.org/2001/XMLSchema"
        xmlns="http://schema061.uri/"
        exclude-result-prefixes="xs">

<!-- PURPOSE: Test validation="strict", including nested validation of child elements
     and substitution groups -->
              <?spec xslt#import-schema?>
              <?spec xslt#validating-using-validation-attribute?>

<xsl:import-schema schema-location="schema061.xsd" 
     namespace="http://schema061.uri/"/>
  



  <xsl:template match="/">
    <address zip="rg4" xsl:validation="strict">
      <person>
        <first>Michael</first>
        <middle>H</middle>
        <surname>Kay</surname>
      </person>
      <age>52</age>
    </address>
  </xsl:template>
  
</xsl:stylesheet>