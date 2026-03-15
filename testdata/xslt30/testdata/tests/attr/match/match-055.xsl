<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:m="http://match85/"
    exclude-result-prefixes="xs m">

<?spec xslt#patterns?>
    <!-- Purpose: Test of element-with-id('literal') as match pattern, new in XSLT 3.0. -->
  
<xsl:import-schema namespace="http://match85/">
  <xs:schema targetNamespace="http://match85/">
    <xs:element name="table">
        <xs:complexType>
            <xs:sequence>
                <xs:element name="row"  maxOccurs="unbounded" form="qualified">
                  <xs:complexType>
                    <xs:sequence>
                      <xs:element name="id" type="xs:ID"  form="qualified"/>
                      <xs:element name="value" type="xs:string"  form="qualified"/>
                    </xs:sequence>
                  </xs:complexType>
                </xs:element>
            </xs:sequence>
        </xs:complexType>
    </xs:element>
  </xs:schema>
</xsl:import-schema> 
  
<xsl:variable name="x" as="document-node()">
    <xsl:document validation="strict" xmlns="http://match85/">
        <table>
            <row><id>A</id><value>ALPHA</value></row>
            <row><id>B</id><value>BETA</value></row>
            <row><id>C</id><value>GAMMA</value></row>
            <row><id>D</id><value>DELTA</value></row>
        </table>
    </xsl:document>
</xsl:variable>  

<xsl:template match="/">
  <out>
    <xsl:apply-templates select="$x//m:row"/>
  </out>
</xsl:template>

<xsl:template match="element-with-id('C')">
  <xsl:copy-of select="."/>
</xsl:template>

<xsl:template match="text()"/>

</xsl:stylesheet>