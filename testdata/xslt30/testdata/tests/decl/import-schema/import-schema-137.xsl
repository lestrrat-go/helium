<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
        xmlns:xs="http://www.w3.org/2001/XMLSchema"
        xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
        xmlns:z="http://schema061.uri/"
        exclude-result-prefixes="xsi">

<!-- PURPOSE: Type-checking of constructed elements involving substitution groups -->
             <?spec xslt#annotation-for-constructed-element?>
              <?error ?>

<xsl:import-schema namespace="http://schema061.uri/" schema-location="schema061.xsd"/>



  <xsl:template match="/">
    <out>
      <xsl:call-template name="p"/>
    </out>
  </xsl:template>
  
  <xsl:template name="p" as="schema-element(z:person)">
    <z:person xsl:validation="strict">
      <xsl:element name="z:title" validation="lax">Dr.</xsl:element>
      <z:first>Michael</z:first>
      <z:middle>H</z:middle>
      <xsl:call-template name="sn"/>
    </z:person>
  </xsl:template>
  
  <xsl:template name="sn">
    <z:familyname xsl:validation="strict">Kay</z:familyname>
  </xsl:template>
  
</xsl:stylesheet>