<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

  <!-- Basic test of xsl:iterate; exercises position() and last() -->


  <xsl:template match="/">
    <out>
      <xsl:iterate select="//ITEM/TITLE">
        <item position="{position()}" last="{last()}">
          <xsl:copy-of select="."/>
        </item>
      </xsl:iterate>
    </out>
  </xsl:template>

</xsl:stylesheet>
