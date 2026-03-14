<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:tns="http://www.3701.com" version="3.0" exclude-result-prefixes="xs">
  
  <xsl:import-schema namespace="http://www.3701.com">
    <xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" elementFormDefault="qualified"
      targetNamespace="http://www.3701.com" xmlns:tns="http://www.3701.com">
      <xs:element name="r">
        <xs:complexType>
          <xs:sequence>
            <xs:element maxOccurs="unbounded" name="t" type="tns:T"/>
          </xs:sequence>
        </xs:complexType>
      </xs:element>
      <xs:complexType name="T">
        <xs:simpleContent>
          <xs:extension base="xs:string">
            <xs:attribute name="id" type="xs:ID"/>
          </xs:extension>
        </xs:simpleContent>
      </xs:complexType>
    </xs:schema>
  </xsl:import-schema>
  
  <xsl:template match="input" as="schema-element(tns:r)">
    <xsl:variable name="tList" as="element(*, tns:T)">
      <xsl:apply-templates select="*"/>
    </xsl:variable>
    <tns:r xsl:validation="strict">
      <xsl:sequence select="$tList"/>
    </tns:r>
  </xsl:template>
  
  <xsl:template match="s" as="element(*, tns:T)">
    <tns:t xsl:type="tns:T">
      <xsl:attribute name="id" select="concat('t-', count(preceding-sibling::*) + 1)"/>
      <xsl:value-of select="."/>
    </tns:t>
  </xsl:template>
  
  <xsl:template name="xsl:initial-template">
    <xsl:variable name="in">
      <input>
        <s>one</s>
      </input>
    </xsl:variable>
    <xsl:apply-templates select="$in"/>
  </xsl:template>
  
</xsl:stylesheet>