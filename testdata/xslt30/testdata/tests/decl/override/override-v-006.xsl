<xsl:package 
  name="http://www.w3.org/xslt30tests/override-v-006"
  package-version="1.0.1"
  version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="xs">
  
  <xsl:import-schema>
    <xs:schema>
      <xs:simpleType name="u6">
        <xs:union memberTypes="xs:dateTime xs:date "/>
      </xs:simpleType>
    </xs:schema>
  </xsl:import-schema>
  
  <xsl:use-package name="http://www.w3.org/xslt30tests/override-base-v-002" package-version="*">
    <xsl:override>
      <xsl:variable name="var" as="u6" select="current-date()" visibility="public"/>
    </xsl:override>
  </xsl:use-package>
  
  <xsl:template name="xsl:initial-template" visibility="public">
    <out><xsl:value-of select="$var instance of xs:date"/></out>
  </xsl:template>

 
  
</xsl:package>   