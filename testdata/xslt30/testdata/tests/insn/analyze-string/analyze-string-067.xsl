<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
	<!-- Purpose: Testcase with xsl:analyze-string using multiline mode (@flags is set to 'm').-->


	<t:template match="/doc">
		<out>

			<E1>
				<t:analyze-string select="poem" regex="Kaum.*krahen">
					<t:matching-substring>
						<t:value-of select="'*'"/>
					</t:matching-substring>
					<t:non-matching-substring>
						<t:value-of select="."/>
					</t:non-matching-substring>
				</t:analyze-string>
			</E1>

			<E2>
				<t:analyze-string select="poem_multiline" regex="Kaum.*krahen">
					<t:matching-substring>
						<t:value-of select="'*'"/>
					</t:matching-substring>
					<t:non-matching-substring>
						<t:value-of select="."/>
					</t:non-matching-substring>
				</t:analyze-string>
			</E2>

			<E3>
				<t:analyze-string select="poem_multiline" regex="Kaum.*krahen" flags="m">
					<t:matching-substring>
						<t:value-of select="'*'"/>
					</t:matching-substring>
					<t:non-matching-substring>
						<t:value-of select="."/>
					</t:non-matching-substring>
				</t:analyze-string>
			</E3>

			<E4>
				<t:analyze-string select="poem" regex="^Kaum.*gesehen,$">
					<t:matching-substring>
						<t:value-of select="'*'"/>
					</t:matching-substring>
					<t:non-matching-substring>
						<t:value-of select="."/>
					</t:non-matching-substring>
				</t:analyze-string>
			</E4>

			<E5>
				<t:analyze-string select="poem_multiline" regex="^Kaum.*gesehen,$">
					<t:matching-substring>
						<t:value-of select="'*'"/>
					</t:matching-substring>
					<t:non-matching-substring>
						<t:value-of select="."/>
					</t:non-matching-substring>
				</t:analyze-string>
			</E5>

			<E6>
				<t:analyze-string select="poem_multiline" regex="^Kaum.*gesehen,$" flags="m">
					<t:matching-substring>
						<t:value-of select="'*'"/>
					</t:matching-substring>
					<t:non-matching-substring>
						<t:value-of select="."/>
					</t:non-matching-substring>
				</t:analyze-string>
			</E6>

		</out>

	</t:template>
</t:transform>
