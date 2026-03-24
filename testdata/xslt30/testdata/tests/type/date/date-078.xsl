<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema"
xmlns:xdt="http://www.w3.org/2005/04/xpath-datatypes"
xmlns:z="http://z.com/" exclude-result-prefixes="xs z xdt">

  <!-- PURPOSE: date subtraction and division of durations -->

    <xsl:param name="d1" as="xs:date" select="xs:date('1900-01-06')"/>


  <xsl:template match="/">
     <out>
     <xsl:for-each select="1 to 10">
       <xsl:variable name="n" select="xs:date('2004-03-01') + (xs:dayTimeDuration('P1D')*.)"/>
       
       <z><xsl:value-of select="$n, ' = ', ($n - xs:date('1901-01-06')), ' # ', ($n - xs:date('1901-01-06')) div xs:dayTimeDuration('P1D') mod 7, '; '"/></z>
     </xsl:for-each>
     </out>
  </xsl:template>

</xsl:stylesheet>
