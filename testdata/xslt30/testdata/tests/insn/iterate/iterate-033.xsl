<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

  <!-- xsl:iterate; initial value of parameter is an empty sequence -->
  <!-- Saxon 9.5 bug report from David Rudel -->

  <xsl:variable name="v">
    <a>
       <session assignment="A"/>
       <session assignment="B"/>
    </a>
  </xsl:variable>

  <xsl:template name="main">
    <out>
       <xsl:iterate select="$v//session">
          <xsl:param name="assignment" as="xs:string?"/>
          <xsl:choose>
             <xsl:when test="not(@assignment eq $assignment)">
                <ok/>
                <xsl:next-iteration>
                   <xsl:with-param name="assignment" select="@assignment"/>
                </xsl:next-iteration>
             </xsl:when>
             <xsl:otherwise>
               <fail/>
             </xsl:otherwise>  
          </xsl:choose>
       </xsl:iterate>
     </out>          
   </xsl:template>
</xsl:stylesheet>
