<?xml version="1.0" encoding="iso-8859-1"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
        xmlns:xs="http://www.w3.org/2001/XMLSchema"
        xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
        xmlns:z="http://schema061.uri/"
        exclude-result-prefixes="xsi">

<!-- PURPOSE: Static type-checking of content of simple-valued elements -->

<xsl:import-schema namespace="http://schema061.uri/" schema-location="schema061.xsd"/>
             <?spec xslt#annotation-for-constructed-element?>
              <?error ?>



  <xsl:template match="/">
    <out>
      <xsl:call-template name="p"/>
    </out>
  </xsl:template>

 
  <xsl:template name="p">
    <z:cash xsl:validation="strict" currency="GBP">£20.00</z:cash>
  </xsl:template>
  

  
  
</xsl:stylesheet>