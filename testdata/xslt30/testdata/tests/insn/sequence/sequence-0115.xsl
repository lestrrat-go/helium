<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- PURPOSE:   Test sequence construction using xsl:sequence with atomization 
    (type check removed 2004-03-25, facility dropped from the spec)-->


<xsl:template match="doc">
  <xsl:variable name="q" as="xs:integer *" xmlns:xs="http://www.w3.org/2001/XMLSchema">
    <xsl:for-each select="1 to 10">
       <xsl:variable name="int" as="xs:integer">
          <e><xsl:value-of select="."/></e>
       </xsl:variable>
       <xsl:sequence select="$int"/>
    </xsl:for-each>
  </xsl:variable>
  <zzz><xsl:value-of select="$q" separator=","/></zzz>
</xsl:template>
 
</xsl:stylesheet>
