<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:map="http://www.w3.org/2005/xpath-functions/map"
  xmlns:err="http://www.w3.org/2005/xqt-errors" xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:f="http://www.w3.org/xslt30tests/functions" exclude-result-prefixes="map xs err f">


  <xsl:strip-space elements="*"/>



  <!-- A shallow-descent function whose first argument is motionless and grounded, and whose
       second argument is grounded and consuming-->
  
  <xsl:function name="f:g"  streamability="shallow-descent">
    <xsl:param name="one" as="node()"/>
    <xsl:param name="two" as="xs:string*"/>
    <xsl:sequence select="if (count($two) gt 1) then $one/chapter/section else ()"/>
  </xsl:function>
  
  <xsl:template name="xsl:initial-template" >
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <xsl:variable name="chap">
        <chapter>
          <section>one</section>
          <section>two</section>
        </chapter>
      </xsl:variable>
      <out>
        <xsl:sequence select="f:g($chap, outermost(//TITLE))"/>
      </out>
    </xsl:source-document>
  </xsl:template>
  
 

</xsl:stylesheet>
