<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:my="http://my.uri/"
exclude-result-prefixes="xs my">

  <!-- PURPOSE: Test as attribute on xsl:function -->
  <?spec xslt#stylesheet-functions?>
  
  <xsl:param name="p" as="xs:date" select="current-date()"/>
  
  <xsl:template match="/">
    <out att="{my:func(xs:byte(1),'x',xs:date('1999-01-09'))}"/>                
  </xsl:template>
  
  <xsl:function name="my:func" as="xs:string*">
     <xsl:param name="a" as="xs:byte"/>
     <xsl:param name="b" as="xs:string"/>
     <xsl:param name="c" as="xs:date"/>
     <xsl:sequence select="(string($a), string($b), string($c))"/>
  </xsl:function>
  

</xsl:stylesheet>