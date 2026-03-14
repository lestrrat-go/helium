<?spec xslt#grouping?>
<xsl:transform version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
   xmlns:xs="http://www.w3.org/2001/XMLSchema">

   <!-- Demonstrate items belonging to zero or more groups, including one that
       has the same grouping key repeated -->

   <xsl:template match="/">
      <out>
         <xsl:for-each-group select="//book" group-by="author">
            <group author="{current-grouping-key()}">
               <xsl:copy-of select="current-group()/title"/>
            </group>
         </xsl:for-each-group>
      </out>
   </xsl:template>

</xsl:transform>
