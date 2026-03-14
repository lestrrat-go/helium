<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <xsl:import-schema schema-location="schema092.xsd"/>
  <!-- test that a namespaced attribute whose namespace is not in scope is present in the validated document -->

  <xsl:template match="/">
      <xsl:variable name="c" as="document-node()">
        <xsl:copy-of select="." validation="strict"/>
      </xsl:variable>  
      <doc>
        <xsl:apply-templates select="$c/doc/e"/>
      </doc> 
  </xsl:template>
  
  <xsl:template match="e">
      <att local="{local-name(@*)}" ns="{namespace-uri(@*)}" 
           prefixed="{contains(name(@*), ':')}"
           declared="{exists(namespace::*[.='http://p.com/'])}">
         <xsl:if test="position()=1 or position()=3">
            <!-- in these cases a prefix is bound and the XSD 1.1 rules for namespace fixup say it must be used -->
            <xsl:attribute name="name" select="name(@*)"/>
         </xsl:if>
      </att>       
  </xsl:template>
  
    

</xsl:stylesheet>