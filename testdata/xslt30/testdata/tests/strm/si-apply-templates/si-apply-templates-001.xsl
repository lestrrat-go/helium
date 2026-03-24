<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                          version="3.0">

   <xsl:output method="xml" />

   <xsl:mode streamable="yes" on-no-match="shallow-copy"/>

   <xsl:template name="main">
       <xsl:source-document streamable="yes" href="../docs/books.xml">
           <Results>
               <xsl:apply-templates select="BOOKLIST/CATEGORIES" />
           </Results>
       </xsl:source-document>
   </xsl:template>

   <xsl:template match="CATEGORY">
       <CATEGORY>
           <Attributes>
               <xsl:for-each select="@*">
                   <Name><xsl:value-of select="name(.)" /></Name>
                   <Value><xsl:value-of select="." /></Value>
               </xsl:for-each>
           </Attributes>
           <Ancestors>
               <xsl:for-each select="ancestor::*">
                   <Name><xsl:value-of select="name(.)" /></Name>
                   <Attributes>
                       <xsl:for-each select="@*">
                           <Name><xsl:value-of select="name(.)" /></Name>
                           <Value><xsl:value-of select="." /></Value>
                       </xsl:for-each>
                   </Attributes>
               </xsl:for-each>
           </Ancestors>
       </CATEGORY>
   </xsl:template>

</xsl:stylesheet>
