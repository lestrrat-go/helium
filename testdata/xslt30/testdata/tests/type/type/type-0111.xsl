<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema"
xmlns:xdt="http://www.w3.org/2005/02/xpath-datatypes" exclude-result-prefixes="xdt">

  <!-- PURPOSE: Test type attribute with parameter passing. -->
  <?spec xslt#with-param?>
  
  <xsl:param name="p" as="xs:date" select="current-date()"/>
  
  <xsl:template match="/">
    <out>;
        <xsl:call-template name="show-params">
          <xsl:with-param name="a" select="xs:byte(100)"/>
          <xsl:with-param name="b" select="xs:dayTimeDuration('PT12H')"/>
          <xsl:with-param name="c" select="xs:yearMonthDuration('P6M')"/>
        </xsl:call-template>                            
    </out> 
  </xsl:template>
  
  <xsl:template name="show-params">
     <xsl:param name="a" as="xs:byte" select="xs:byte(10)"/>
     <xsl:param name="b" as="xs:duration" select="xs:duration('P3Y')"/>
     <xsl:param name="c" as="xs:yearMonthDuration" select="xs:yearMonthDuration('P18M')"/>
     <xsl:variable name="z" as="xs:yearMonthDuration" select="xs:yearMonthDuration('P6M')"/>
     <q>a=<xsl:value-of select="$a"/></q>;
     <q>b=<xsl:value-of select="$b"/></q>;
     <q>c+z=<xsl:value-of select="$c + $z"/></q>;
  </xsl:template>
  

</xsl:stylesheet>