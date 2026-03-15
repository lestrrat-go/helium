<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
            xmlns:bdd="http://bdd.test.com">

 <?spec xslt#creating-attributes?>
    <!-- Purpose: The attribute must be in the designated namespace, even if the prefix has
    to be reset or ignored. -->

<xsl:template match="/">
  <out>
    <xsl:variable name="jam" as="element()">
    <bdd:jam>
      <xsl:attribute name="{docs/b}" namespace="http://xyz.com">jaminben</xsl:attribute>

    </bdd:jam>
    </xsl:variable>
    <ans local="{local-name($jam/@*)}" ns="{namespace-uri($jam/@*)}" 
         good-prefix="{not(starts-with(name($jam/@*),'bdd:'))}"/>
  </out>
</xsl:template>

</xsl:stylesheet>
