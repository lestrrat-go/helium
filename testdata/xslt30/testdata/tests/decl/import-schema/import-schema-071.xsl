<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
        xmlns:xs="http://www.w3.org/2001/XMLSchema"
        xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
        xmlns:z="http://schema081.uri/"
        exclude-result-prefixes="xsi">

<!-- PURPOSE: Test use of an inline schema with an xs:include -->
<?spec xslt#import-schema?>

<xsl:import-schema namespace="http://schema081.uri/">
  <xs:schema targetNamespace="http://schema081.uri/">
    
    <xs:include schemaLocation="xschema081inc.xsd"/>
    
    <xs:simpleType name="yesOrNo">
      <xs:restriction base="xs:string">
        <xs:enumeration value="yes"/>
        <xs:enumeration value="no"/>
      </xs:restriction>
    </xs:simpleType>
  </xs:schema>
</xsl:import-schema>





  <xsl:template match="/">
    <out>
      <xsl:attribute name="do" type="z:blackOrWhite">white</xsl:attribute>
    </out>
  </xsl:template>
  
  
</xsl:stylesheet>