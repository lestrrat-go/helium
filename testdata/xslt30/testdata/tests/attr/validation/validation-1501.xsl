<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" version="2.0">

  <xsl:import-schema>
    <xs:schema>
      <xs:element name="data">
        <xs:complexType>
          <xs:sequence>
            <xs:element ref="events"/>
            <xs:element ref="participants"/>
          </xs:sequence>
        </xs:complexType>
      </xs:element>
      <xs:element name="events">
        <xs:complexType>
          <xs:sequence>
            <xs:element ref="event" maxOccurs="unbounded"/>
          </xs:sequence>
        </xs:complexType>
      </xs:element>
      <xs:element name="participants">
        <xs:complexType>
          <xs:sequence>
            <xs:element ref="participant" maxOccurs="unbounded"/>
          </xs:sequence>
        </xs:complexType>
      </xs:element>
      <xs:element name="event">
        <xs:complexType>
          <xs:attribute name="id" type="xs:ID"/>
          <xs:attribute name="name" type="xs:string"/>
        </xs:complexType>
      </xs:element>
      <xs:element name="participant">
        <xs:complexType>
          <xs:attribute name="events" type="xs:IDREFS"/>
          <xs:attribute name="name" type="xs:string"/>
        </xs:complexType>
      </xs:element>
    </xs:schema>
  </xsl:import-schema>

  <xsl:template match="/">
    <out>
      <xsl:variable name="p" as="document-node(schema-element(data))">
        <xsl:copy-of select="." validation="strict"/>
      </xsl:variable>
      <xsl:for-each select="$p/data/events/event">
        <i>
          <xsl:copy-of select="idref(@id)/.."/>
        </i>
      </xsl:for-each>
    </out>
  </xsl:template>

</xsl:stylesheet>
