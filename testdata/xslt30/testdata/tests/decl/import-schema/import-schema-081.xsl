<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
        xmlns:xs="http://www.w3.org/2001/XMLSchema"
        xmlns="http://schema091.uri/"
        xmlns:t="http://schema091.uri/"
        exclude-result-prefixes="xs t"
        default-validation="preserve">

<!-- PURPOSE: Test use of base URI in an inline schema -->
<?spec xslt#import-schema?>

<xsl:import-schema namespace="http://schema091.uri/" xml:base="dir/">
<xs:schema targetNamespace="http://schema091.uri/"
           elementFormDefault="qualified">
<xs:import namespace="http://schema091a.uri/" schemaLocation="schema091a.xsd"/>
<xs:element name="money" type="a:money-amount" xmlns:a="http://schema091a.uri/"/>
  
</xs:schema>
</xsl:import-schema>
  



  <xsl:template match="/">
    <t:money validation="strict">23.50</t:money>
  </xsl:template>
  
</xsl:stylesheet>