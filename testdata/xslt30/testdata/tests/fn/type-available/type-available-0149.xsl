<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs" version="2.0">

  <!-- Test type-available() - availability of user-defined types with a schema-aware processor  -->

  <xsl:import-schema>
    <xs:schema>
      <xs:simpleType name="shortString">
        <xs:restriction base="xs:string">
          <xs:maxLength value="5"/>
        </xs:restriction>
      </xs:simpleType>
      <xs:complexType name="monotony">
        <xs:sequence minOccurs="0" maxOccurs="unbounded">
          <xs:element name="a"/>
        </xs:sequence>
      </xs:complexType>
      <xs:simpleType name="list">
        <xs:list itemType="xs:integer"/>
      </xs:simpleType>
      <xs:simpleType name="union">
        <xs:union memberTypes="xs:gYear xs:gYearMonth"/>
      </xs:simpleType>
    </xs:schema>
  </xsl:import-schema>


  <xsl:template match="/">
    <out>
      <a>
        <xsl:value-of select="type-available('shortString')"/>
      </a>
      <a>
        <xsl:value-of select="type-available('monotony')"/>
      </a>
      <a>
        <xsl:value-of select="type-available('list')"/>
      </a>
      <a>
        <xsl:value-of select="type-available('union')"/>
      </a>
    </out>
  </xsl:template>

</xsl:transform>
