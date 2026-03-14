<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
        xmlns:xs="http://www.w3.org/2001/XMLSchema"
        xmlns:enu="http://schema050.uri/"
        exclude-result-prefixes="xs enu">

<!-- PURPOSE: Test a (rather simple) complex type -->

    <?spec xslt#validation-xsl-type?>
     <?error ?>

<xsl:import-schema schema-location="schema050.xsd" 
     namespace="http://schema050.uri/"/>
  



  <xsl:template match="/">
    <enu:out>
    <z xsl:type="enu:abf"><enu:f/></z>         
    </enu:out>
  </xsl:template>
  
</xsl:stylesheet>