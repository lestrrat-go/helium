<?xml version="1.0"?> 

<!-- try005 -->

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
xmlns:err="http://www.w3.org/2005/xqt-errors"
xmlns:f="http://local-functions/"
xmlns:xs="http://www.w3.org/2001/XMLSchema"
xmlns:z="http://try005.com/"
exclude-result-prefixes="f xs z err"
>

<!-- xsl:try() around element creation should not catch an error in validating the element's parent -->

<xsl:output indent="yes"/>

<xsl:import-schema>
<xs:schema targetNamespace="http://try005.com/">
  <xs:complexType name="z">
    <xs:sequence>
      <xs:element name="a"/>
      <xs:element name="b"/>
      <xs:element name="c"/>
    </xs:sequence>
  </xs:complexType>
</xs:schema>
</xsl:import-schema>


<xsl:template name="main">
  <!-- Under the changes agreed since the July 2012 draft, the validation error should not be caught-->
  <o><z xsl:type="z:z">
  <xsl:try select="f:run()">
    <xsl:catch errors="*">
      <validation-error>
        <err:error code="{$err:code}"
             module="{$err:module}"
             line="{$err:line-number}">
           <message><xsl:value-of select="$err:description"/></message>
        </err:error>
      </validation-error> 
    </xsl:catch>
  </xsl:try>
  </z>
  </o>
</xsl:template>

<xsl:function name="f:run">
    <a/>
    <d/>
 </xsl:function>

</xsl:stylesheet>
