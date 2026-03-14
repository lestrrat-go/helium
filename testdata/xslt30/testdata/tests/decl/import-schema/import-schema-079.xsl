<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
        xmlns:xs="http://www.w3.org/2001/XMLSchema"
        xmlns="http://schema089.uri/"
        xmlns:t="http://schema089.uri/"
        exclude-result-prefixes="xs t"
        default-validation="preserve">

<!-- PURPOSE: Test that a subtype of xs:ID is recognized for element nodes -->
<?spec xslt#import-schema?>
<?spec fo#func-id?>

<xsl:import-schema>
    <!-- defaults the namespace attribute - see erratum E9 -->
<xs:schema targetNamespace="http://schema089.uri/"
           elementFormDefault="qualified">

<xs:simpleType name="tetra">
    <xs:restriction base="xs:ID">
      <xs:pattern value="[a-z]{4}"/>
    </xs:restriction>
</xs:simpleType>

</xs:schema>
</xsl:import-schema>
  



  <xsl:template match="/">
    <xsl:variable name="n">
    <out>
      <person>
        <xsl:element name="code" type="t:tetra">asdf</xsl:element>
        <first>Michael</first>
        <middle>H</middle>
        <last>Kay</last>
      </person>
      <person>
        <xsl:element name="code" type="t:tetra">gh<xsl:comment>1234</xsl:comment>jk</xsl:element>
        <first>Penny</first>
        <surname>Whelan</surname>
      </person>
      <person>
      <xsl:element name="code" type="t:tetra">bnmp</xsl:element>
        <last>Kay</last>
        <first>Philippa</first>
        <middle>H</middle>
      </person> 
    </out>
    </xsl:variable>
    <ans>
      <xsl:copy-of select="$n/id('ghjk')/.."/>               
    </ans>       
  </xsl:template>
  
</xsl:stylesheet>