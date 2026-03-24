<?xml version="1.0" encoding="UTF-8"?>

<!-- show schema declarations are available if schema-aware = yes -->

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:ev28="http://example.com/evaluate028" version="3.0"
exclude-result-prefixes="xs ev28">

  <xsl:output method="xml" encoding="UTF-8"/>
  
  <xsl:import-schema>
    <xs:schema targetNamespace="http://example.com/evaluate028">
      <xs:simpleType name="restrictedDate">
        <xs:restriction base="xs:date">
          <xs:minInclusive value="2009-01-01"/>
        </xs:restriction>
      </xs:simpleType>
    </xs:schema>
  </xsl:import-schema>
  
  <xsl:param name="path">$val instance of ev28:restrictedDate</xsl:param>
  
  <xsl:template match="/" name="main">
    <Result>
      <xsl:variable name="now" select="ev28:restrictedDate(current-date())"/>
      <value>
         <xsl:evaluate xpath="$path" as="xs:boolean" schema-aware="yes">
           <xsl:with-param name="val" select="$now"/>
         </xsl:evaluate>
      </value>
    </Result>
  </xsl:template>
  
</xsl:stylesheet> 
