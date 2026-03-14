<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

                            <!--It is a non-recoverable
                                dynamic error if the conflict resolution algorithm for
                                template rules leaves more than one matching template rule  
                                when the declaration of the relevant mode has an
                                on-multiple-match attribute with the value fail.-->
   <xsl:template name="main">
      <out>
         <xsl:apply-templates select="$a"/>
      </out>
   </xsl:template>
   
   <xsl:variable name="a" as="element()"><a><b/></a></xsl:variable>
   
   <xsl:mode on-multiple-match="fail"/>
   
   <xsl:template match="a[b]"><x/></xsl:template>
   
   <xsl:template match="a[*]"><x/></xsl:template>
   
</xsl:stylesheet>
