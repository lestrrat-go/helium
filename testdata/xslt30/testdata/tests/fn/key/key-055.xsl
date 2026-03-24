<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

   <!-- A classic 2-level Muenchian Grouping example -->


   <xsl:key name="domain" match="@domain" use="."/>
   <xsl:key name="tech" match="TECHNOLOGY" use="concat(., '/', @domain)"/>
   
   <xsl:strip-space elements="*"/>

   <xsl:template match="/">
      <a>
         <xsl:for-each select="//@domain[generate-id(.)=generate-id(key('domain',.)[1])]">
            <!-- for each distinct domain -->
            <xsl:text>&#xa;</xsl:text>
            <b>Domain <xsl:value-of select="."/>
               <xsl:variable name="dom" select="."/>
               <xsl:for-each
                  select="//TECHNOLOGY[@domain=current()]
            [generate-id(.)=generate-id(key('tech', concat(., '/', @domain))[1])]">
                  <c>Technology <xsl:value-of select="."/>
                     <xsl:for-each
                        select="/PRODUCTS/PRODUCT[TECHNOLOGY[@domain=$dom and .=current()]]">
                        <d>Product <xsl:value-of select="NAME"/></d>
                     </xsl:for-each></c>
               </xsl:for-each></b>
         </xsl:for-each>
      </a>

   </xsl:template>
</xsl:stylesheet>
