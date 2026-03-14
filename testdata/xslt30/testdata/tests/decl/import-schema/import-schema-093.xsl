<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
        xmlns:xs="http://www.w3.org/2001/XMLSchema"
        xmlns:enu="http://schema051.uri/"
        exclude-result-prefixes="xs enu">

<!-- PURPOSE: Test a (rather simple) complex type involving a choice -->

    <?spec xslt#validation-xsl-type?>
     <?error ?>

<xsl:import-schema schema-location="schema051.xsd" 
     namespace="http://schema051.uri/"/>
  



  <xsl:template match="/">
    <enu:out>
    <z xsl:type="enu:choice-of-3"><enu:s/></z>         
    </enu:out>
  </xsl:template>
  
</xsl:stylesheet>