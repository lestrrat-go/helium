<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
               xmlns:f="http://www.w3.org/2005/Atom"
               xmlns:xs="http://www.w3.org/2001/XMLSchema"
               version="2.0">

   <!-- This stylesheet is a repro of Saxon bug 2123. It is designed to check
     that the in-scope namespaces of a copied node are correct. The particular
     use of a pseudo-recursive function to create the copy is designed to take
     Saxon down a particular path where the copy is a virtual copy of the original
     node rather than a physical copy -->

   <xsl:template match="/">
      <p><xsl:value-of select="f:firstType(.,1)/in-scope-prefixes(.)" separator="|"/></p>
   </xsl:template>
   
   <xsl:function name="f:firstType">
     <xsl:param name="doc" as="node()"/>
     <xsl:param name="which" as="xs:integer"/>
     <xsl:copy-of select="($doc/*/*)[$which]"/>
     <xsl:if test="$which lt 100">
       <xsl:sequence select="f:firstType($doc, 120)"/>
     </xsl:if>  
   </xsl:function>

</xsl:stylesheet>