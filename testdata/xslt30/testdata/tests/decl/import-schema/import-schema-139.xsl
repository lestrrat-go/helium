<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
        xmlns:xs="http://www.w3.org/2001/XMLSchema"
        xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
        xmlns:z="http://schema061.uri/"
        exclude-result-prefixes="xsi">

<!-- PURPOSE: Static type-checking of content of simple-valued elements -->

     
             <?spec xslt#annotation-for-constructed-element?>
              <?error ?>

<xsl:import-schema namespace="http://schema061.uri/" schema-location="schema061.xsd"/>



  <xsl:template match="/">
    <out>
      <xsl:call-template name="sn"/>
    </out>
  </xsl:template>

  <xsl:template name="sn">
    <money xsl:type="z:currency-type">CHF</money>
  </xsl:template>
  


  
  
</xsl:stylesheet>