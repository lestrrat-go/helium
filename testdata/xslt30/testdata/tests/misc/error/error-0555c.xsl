<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">            

                    <!--It is a non-recoverable dynamic error if xsl:apply-templates,
                        xsl:apply-imports or xsl:next-match is
                        used to process a node using a mode whose declaration specifies
                        on-no-match="fail" when there is no template rule in the stylesheet whose
                        match pattern matches that node. -->
   
   <xsl:mode on-no-match="fail"/>
   
   <xsl:variable name="v" as="element()"><q/></xsl:variable>
   
   <xsl:template name="main">
      <out>
         <xsl:apply-templates select="$v"/>
      </out>
   </xsl:template>
   
   <xsl:template match="q">
     <q><xsl:apply-imports/></q>
   </xsl:template>
   
</xsl:stylesheet>
