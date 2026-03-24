<xsl:package 
  name="http://www.w3.org/xslt30tests/override-base-v-002"
  package-version="1.0.1"
  version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="xs">
  
  <xsl:import-schema>
    <xs:schema>
      <xs:simpleType name="u1">
        <xs:union memberTypes="xs:date xs:time xs:dateTime"/>
      </xs:simpleType>
    </xs:schema>
  </xsl:import-schema>
  
  <xsl:variable name="var" as="u1" select="current-dateTime()" visibility="public"/>
  

 
  
</xsl:package>   