<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs">

<?spec xslt#patterns?>
    <!-- Purpose: Test of id('literal') with id-valued element as match pattern -->
  
<xsl:import-schema>
  <xs:schema>
    <xs:element name="table">
        <xs:complexType>
            <xs:sequence>
                <xs:element name="row"  maxOccurs="unbounded">
                  <xs:complexType>
                    <xs:sequence>
                      <xs:element name="id" type="xs:ID"/>
                      <xs:element name="value" type="xs:string"/>
                    </xs:sequence>
                  </xs:complexType>
                </xs:element>
            </xs:sequence>
        </xs:complexType>
    </xs:element>
  </xs:schema>
</xsl:import-schema> 
  
<xsl:variable name="x" as="document-node()">
    <xsl:document validation="strict">
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
    <xsl:apply-templates select="$x//row"/>
  </out>
</xsl:template>

<xsl:template match="id('C')">
  <xsl:copy-of select=".."/>
</xsl:template>

<xsl:template match="text()"/>

</xsl:stylesheet>