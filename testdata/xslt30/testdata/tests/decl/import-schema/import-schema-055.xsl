<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
        xmlns:xs="http://www.w3.org/2001/XMLSchema"
        xmlns="http://schema061.uri/"
        xpath-default-namespace="http://schema061.uri/"
        exclude-result-prefixes="xs">

<!-- PURPOSE: Test document-node(element(x,y)) node tests -->

<xsl:import-schema schema-location="schema061.xsd" 
     namespace="http://schema061.uri/"/>
              <?spec xslt#import-schema?>
              <?spec xslt#validating-using-validation-attribute?>
              <?spec xpath#id-matching-item?>
  



<xsl:param name="address">address</xsl:param>
<xsl:param name="person">person</xsl:param>
<xsl:param name="first">first</xsl:param>
<xsl:param name="middle">middle</xsl:param>
<xsl:param name="last">last</xsl:param>
<xsl:param name="age">age</xsl:param>
<xsl:param name="zip">zip</xsl:param>

  <xsl:template match="/*">
  <xsl:variable name="t">
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
  <t><xsl:value-of select="$t instance of document-node(schema-element(address))"/></t>
  <t><xsl:value-of select="$t instance of document-node(element(address))"/></t>
  <t><xsl:value-of select="$t instance of document-node(element(address,addressType))"/></t> 
  <xsl:variable name="u">
    <xsl:copy-of select="$t/*" validation="preserve"/>
    <xsl:copy-of select="$t/*" validation="preserve"/>
  </xsl:variable>
  <f><xsl:value-of select="$u instance of document-node(schema-element(address))"/></f>
  <apply><xsl:apply-templates select="$t" mode="test"/></apply>
  <apply><xsl:apply-templates select="$u" mode="test"/></apply>  
  </out>
  </xsl:template>
  
  <xsl:template match="document-node(schema-element(address))" mode="test">yes</xsl:template>
  <xsl:template match="/" mode="test">no</xsl:template>
  
</xsl:stylesheet>