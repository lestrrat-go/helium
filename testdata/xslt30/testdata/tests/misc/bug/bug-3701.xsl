<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

<?spec xslt#key?>
<!-- BUG: Saxon 5.5/001 -->
<!-- DESCRIPTION:
    An error occurs when a filter expression using the key() function is used as an argument
    to another function call within an XPath predicate. -->

  <xsl:key name="files-in-chapter" match="file"  use="@found-in"/>
  <xsl:key name="chapter-for-files" match="file" use="@name"/>
  <xsl:key name="distinct" match="file/@found-in" use="."/>
  <xsl:strip-space elements="*"/>

  <xsl:template match="/">
    <crossref>
      <xsl:for-each select="//@found-in[generate-id()
                                        =generate-id(key('distinct',.)[1])]"
                                        >
        <xsl:sort select="."/>
        <chapter id="{.}">
          <xsl:for-each select="key('files-in-chapter',.)">
            <name><xsl:value-of select="@name"/></name>
          </xsl:for-each>
        </chapter>
      </xsl:for-each>
    </crossref>
  </xsl:template>
</xsl:stylesheet>
