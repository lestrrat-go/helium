<?xml version="1.0"?> 

<!-- try004 -->

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
xmlns:f="http://local-functions/"
xmlns:xs="http://www.w3.org/2001/XMLSchema"
xmlns:err="http://www.w3.org/2005/xqt-errors"
xmlns:z="http://try004.com/"
exclude-result-prefixes="f xs z err"
>

<!-- Use xsl:try to catch an error doing output validation -->

<xsl:import-schema>
<xs:schema targetNamespace="http://try004.com/">
  <xs:complexType name="z">
    <xs:sequence>
      <xs:element name="a"/>
      <xs:element name="b"/>
      <xs:element name="c"/>
    </xs:sequence>
  </xs:complexType>
</xs:schema>
</xsl:import-schema>

<xsl:param name="d" select="'d'"/>


<xsl:template name="main">
  <o><xsl:try select="f:run()">
    <xsl:catch errors="*">
      <validation-error>
        <err:error code="{$err:code}"
             module="{tokenize($err:module, '/')[last()]}"
             line="{$err:line-number idiv 1000}">
           <message><xsl:value-of select="boolean($err:description)"/></message>
        </err:error>
      </validation-error> 
    </xsl:catch>
  </xsl:try>
  </o>
</xsl:template>

<xsl:function name="f:run">
  <z xsl:type="z:z">
    <a/>
    <xsl:element name="{$d}"/>
  </z>
</xsl:function>

</xsl:stylesheet>
