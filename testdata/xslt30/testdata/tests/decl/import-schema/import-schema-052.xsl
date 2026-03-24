<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
        xmlns:xs="http://www.w3.org/2001/XMLSchema"
        xmlns="http://schema061.uri/"
        xpath-default-namespace="http://schema061.uri/"
        exclude-result-prefixes="xs">

<!-- PURPOSE: Test type annotations on validated elements with complex type -->
              <?spec xslt#import-schema?>
              <?spec xslt#validating-using-validation-attribute?>
              <?spec xpath#id-element-test?>
              <?spec xpath#id-schema-element-test?>

<xsl:import-schema schema-location="schema061.xsd" 
     namespace="http://schema061.uri/"/>
  



  <xsl:template match="/">
  <xsl:variable name="t" as="element()">
    <address zip="rg4" xsl:validation="strict">
      <person>
        <first>Michael</first>
        <middle>H</middle>
        <last>Kay</last>
      </person>
      <age>52</age>
    </address>
  </xsl:variable>
  <out>
  <t><xsl:value-of select="$t instance of schema-element(address)"/></t>
  <t><xsl:value-of select="$t instance of element(address)"/></t>
  <t><xsl:value-of select="$t instance of element(address,addressType)"/></t>
  <t><xsl:value-of select="$t instance of element(*,addressType)"/></t>
  <t><xsl:value-of select="$t/person instance of schema-element(person)"/></t>   
  <t><xsl:value-of select="$t/person instance of element(person)"/></t>
  <t><xsl:value-of select="$t/person/middle instance of element(*,xs:string)"/></t>
  <f><xsl:value-of select="$t instance of schema-element(person)"/></f>
  <f><xsl:value-of select="$t instance of element(person)"/></f>
  <f><xsl:value-of select="$t instance of element(address,xs:string)"/></f>
  <f><xsl:value-of select="$t instance of element(*,xs:string)"/></f>
  <f><xsl:value-of select="$t/person instance of schema-element(address)"/></f>   
  <f><xsl:value-of select="$t/person instance of element(address)"/></f> 
  <f><xsl:value-of select="$t/person/middle instance of element(*,addressType)"/></f>
  </out>         
  </xsl:template>
  
</xsl:stylesheet>