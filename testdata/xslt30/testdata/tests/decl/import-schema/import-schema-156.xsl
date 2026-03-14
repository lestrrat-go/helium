<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
        xmlns:xs="http://www.w3.org/2001/XMLSchema-wrong-namespace"
        xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
        xmlns:z="http://schema080.uri/"
        exclude-result-prefixes="xsi">

<!-- PURPOSE: Test invalid inline schema -->
              <?spec xslt#import-schema?>
              <?error XTSE0220 XTSE0010?> <!-- see bug 893 -->

<xsl:import-schema namespace="http://schema080.uri/">
  <xs:schema targetNamespace="http://schema080.uri/">
    <xs:simpleType name="yesOrNo">
      <xs:restriction base="xs:string">
        <xs:enumeration values="yes"/>
        <xs:enumeration values="no"/>
      </xs:restriction>
    </xs:simpleType>
  </xs:schema>
</xsl:import-schema>





  <xsl:template match="/">
    <out>
      <xsl:attribute name="do" type="z:yesOrNo">maybe</xsl:attribute>
    </out>
  </xsl:template>
  
  
</xsl:stylesheet>