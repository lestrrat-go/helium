<?xml version="1.0"?> 
<xsl:stylesheet 
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  version="2.0">

  <!-- Test xsl:number using circled digits -->
  
  <xsl:param name="from" required="yes" as="xs:integer" />
  <xsl:param name="to" required="yes" as="xs:integer"/>
  <xsl:param name="format-string" required="yes" as="xs:string" />
  
  <xsl:template match="/">
    <out>
      <xsl:for-each select="$from to $to">
        <xsl:number value="." format="{$format-string}"/>
      </xsl:for-each>
    </out>
  </xsl:template>
      

</xsl:stylesheet>
