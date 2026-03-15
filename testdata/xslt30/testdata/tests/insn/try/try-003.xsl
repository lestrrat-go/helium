<?xml version="1.0"?> 

<!-- try003 (was saxon008) -->

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
xmlns:f="http://localhost/functions"
xmlns:xs="http://www.w3.org/2001/XMLSchema"
xmlns:err="http://www.w3.org/2005/xqt-errors"
extension-element-prefixes=""
exclude-result-prefixes="f xs err"
>

<!-- Test xsl:try with a selective catch. The catch rethrows this particular error -->

<xsl:output indent="yes"/>
<xsl:param name="in" as="xs:string" select="'1'"/>

<xsl:template name="main">
  <xsl:message>** Should report "divide by zero" error **</xsl:message>
  <o>
  <xsl:try select="xs:integer($in) div 0">
    <xsl:catch errors="err:FORG0001">
       <error code="{$err:code}" line="{$err:line-number}"/>
    </xsl:catch>
    <xsl:catch errors="*" select="error($err:code, $err:description, $err:value)"/>
  </xsl:try>
  </o>
</xsl:template>


</xsl:stylesheet>
