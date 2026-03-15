<?xml version="1.0" encoding="iso-8859-1" ?>
<?spec xslt#sorting?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
<xsl:output method="xml"/>

  <!-- FileName: sort36 -->
  <!-- Document: http://www.w3.org/TR/xslt -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 10 -->
    <!-- Purpose: Test the famous technique for sorting with conditionals.
     In this case, we want to sort strings with and without "Re: " prefix by
     the rest of the string. Technique from Oliver Becker (obecker@informatik.hu-berlin.de). -->

<xsl:template match="/">
  <out><xsl:text>Memos by thread:
</xsl:text>
    <xsl:for-each select="doc/memo">
      <xsl:sort select="concat(
        substring(subj,1,number(not(starts-with(.,'Re: ')))*string-length(subj)),
        substring(substring-after(subj,'Re: '),1,
          number(starts-with(.,'Re: '))*string-length(substring-after(subj,'Re: '))))"
        data-type="text" />
        <!-- First substring in concat is null string when 'Re: ' is at start, all of subj otherwise.
          Second substring in concat is null at opposite times from first, when non-null it's all of subj beyond 'Re: '.
          Thus, concat "chooses" which string (all of subj or trimmed subj) based on the boolean starts-with(.,'Re: '),
          because one of the two substring functions returns a null string while other doesn't.
          The substring(x,1,0) returns null and substring(x,1,string-length(x)) returns the whole x.
          The zero comes from converting the boolean false to a number, where multiplying by string-length stays zero.
          The other branch gets boolean true converted to 1, then multiplied by string-length. -->
      <xsl:sort select="time" data-type="text" /><!-- Tie breaker, for demonstration only. -->
      <xsl:value-of select="body"/><xsl:text>: </xsl:text><xsl:value-of select="subj"/><xsl:text>;
</xsl:text>
    </xsl:for-each>
  </out>
</xsl:template>

</xsl:stylesheet>